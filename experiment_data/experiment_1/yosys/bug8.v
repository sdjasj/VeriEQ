`timescale 1ns/1ps

module top (
    input  wire signed [21:21] in0,
    input        clock_0,
    input        clock_2
);

    wire clock_0;
    wire clock_2;
    reg signed [28:26] reg_10;

    always @(posedge clock_0 or posedge clock_2) begin
        $display("always block is triggered......");

        if (clock_2) begin
            $display("now in clock_2 == true branch");
        end else begin
            $display("now in clock_2 == false branch");

            if (in0) begin
                reg_10[28:26] <= 2;
            end else begin
                reg_10[28:27] <= 1;
            end
        end
    end

endmodule
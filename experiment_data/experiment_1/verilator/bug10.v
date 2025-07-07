`timescale 1ns/1ps
module top (out81);

    reg [2:0] in2;
    output wire  [24:23] out81;

    assign out81 = {3000{in2[1:0]}} / {2000{1'b1}}; // `%` works the same way.

    initial begin
        in2 = 3;
        #10;
        $display(out81);
    end

endmodule
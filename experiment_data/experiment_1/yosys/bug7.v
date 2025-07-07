module a(input g, c, d, output reg b);
    always @(*) begin
        b = g + d;
    end

    always @(*) begin
        b = c + d;
    end
endmodule
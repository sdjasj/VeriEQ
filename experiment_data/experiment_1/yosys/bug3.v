module a(input b, input [83 : 10] c, d, e, output f);
    reg g;
    always @(*) begin
        f <= g;
        g = d / b;
        if (d / g) begin
            g = b ? b / e / c : 0;
        end
    end 
endmodule